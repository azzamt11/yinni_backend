package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"yinni_backend/internal/conf"

	"yinni_backend/ent"
	"yinni_backend/ent/migrate"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "github.com/go-sql-driver/mysql"
	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string = "product-service"
	// Version is the version of the compiled software.
	Version string = "v1.0.0"
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

var initDB bool

func init() {
	flag.BoolVar(&initDB, "init", false, "run database initialization")
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()

	// // Check for CLI commands (migrate, seed, etc.)
	// if len(os.Args) > 1 && os.Args[1] == "init" {
	// 	runInitialization()
	// 	return
	// }

	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	app, cleanup, err := wireApp(bc.Server, bc.Auth, bc.Data, bc.Embeddings, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// Run database initialization in background before starting the server
	// go runBackgroundInitialization(bc.Data, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := initializeDatabase(ctx, bc.Data, logger); err != nil {
		panic(err)
	}

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}

// runBackgroundInitialization runs database initialization in the background
func runBackgroundInitialization(conf *conf.Data, logger log.Logger) {
	logHelper := log.NewHelper(logger)
	logHelper.Info("Starting background database initialization...")

	// Give the server a moment to start up
	time.Sleep(3 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := initializeDatabase(ctx, conf, logger); err != nil {
		logHelper.Error("Database initialization failed: ", err)
	} else {
		logHelper.Info("Database initialization completed successfully")
	}
}

// runInitialization runs initialization as a CLI command
func runInitialization() {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	logHelper := log.NewHelper(logger)

	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := initializeDatabase(ctx, bc.Data, logger); err != nil {
		logHelper.Error("Initialization failed: ", err)
		os.Exit(1)
	}

	logHelper.Info("Initialization completed successfully")
	os.Exit(0)
}

// initializeDatabase is the shared initialization logic
func initializeDatabase(ctx context.Context, conf *conf.Data, logger log.Logger) error {
	logHelper := log.NewHelper(logger)
	logHelper.Info("Running database initialization...")

	// Connect to database
	drv, err := sql.Open(dialect.MySQL, conf.Database.Source)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Get the underlying sql.DB for ping
	db := drv.DB()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	// Wait for database
	logHelper.Info("Waiting for database to be ready...")
	start := time.Now()
	for {
		if time.Since(start) > 30*time.Second {
			return fmt.Errorf("database not ready after 30 seconds")
		}

		if err := db.PingContext(ctx); err == nil {
			break
		}

		logHelper.Info("Database not ready, waiting...")
		time.Sleep(2 * time.Second)
	}

	// Run migrations
	logHelper.Info("Running migrations...")
	if err := client.Schema.Create(
		ctx,
		migrate.WithForeignKeys(true),
		migrate.WithDropIndex(false),
		migrate.WithDropColumn(false),
	); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	logHelper.Info("Migrations completed")

	// Check if we should seed
	count, err := client.Product.Query().Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to check product count: %w", err)
	}

	if count > 0 {
		logHelper.Infof("Database already has %d products, skipping seeding", count)
		return nil
	}

	// Seed data
	logHelper.Info("Seeding database...")
	if err := seedDatabase(ctx, client, logger); err != nil {
		return fmt.Errorf("seeding failed: %w", err)
	}

	logHelper.Info("Database initialization completed successfully")
	return nil
}

// seedDatabase seeds the database with sample data
func seedDatabase(ctx context.Context, client *ent.Client, logger log.Logger) error {
	logHelper := log.NewHelper(logger)

	// Try to load dataset
	dataset, err := loadDataset()
	if err != nil {
		logHelper.Warn("Failed to load dataset: ", err, ", using sample data")
		return seedSampleData(ctx, client, logger)
	}

	return seedFromDataset(ctx, client, dataset, logger)
}

// loadDataset loads the product dataset
func loadDataset() ([]map[string]interface{}, error) {
	paths := []string{
		"/data/products.json",
		"./product_dataset.json",
		"./data/products.json",
		"/app/data/products.json",
	}

	var file *os.File
	var err error

	for _, path := range paths {
		file, err = os.Open(path)
		if err == nil {
			defer file.Close()
			break
		}
	}

	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var dataset []map[string]interface{}
	if err := json.Unmarshal(content, &dataset); err != nil {
		return nil, err
	}

	return dataset, nil
}

// seedFromDataset seeds from the loaded dataset
func seedFromDataset(ctx context.Context, client *ent.Client, dataset []map[string]interface{}, logger log.Logger) error {
	logHelper := log.NewHelper(logger)
	logHelper.Infof("Seeding %d products from dataset", len(dataset))

	// Limit to 1000 products for initial seeding
	maxProducts := 1000
	if len(dataset) > maxProducts {
		dataset = dataset[:maxProducts]
	}

	bulk := make([]*ent.ProductCreate, 0, len(dataset))

	for i, data := range dataset {
		create := client.Product.Create()

		// Set fields from dataset
		if title, ok := data["title"].(string); ok {
			create.SetTitle(title)
		}
		if brand, ok := data["brand"].(string); ok {
			create.SetBrand(brand)
		}
		if category, ok := data["category"].(string); ok {
			create.SetCategory(category)
		}
		if subCategory, ok := data["sub_category"].(string); ok {
			create.SetSubCategory(subCategory)
		}
		if desc, ok := data["description"].(string); ok {
			create.SetDescription(desc)
		}
		if price, ok := data["selling_price"].(string); ok {
			create.SetSellingPrice(price)
			create.SetPriceNumeric(parsePrice(price))
		}
		if actualPrice, ok := data["actual_price"].(string); ok {
			create.SetActualPrice(actualPrice)
		}
		if discount, ok := data["discount"].(string); ok {
			create.SetDiscount(discount)
		}
		if pid, ok := data["pid"].(string); ok {
			create.SetPid(pid)
		}
		if originalID, ok := data["_id"].(string); ok {
			create.SetOriginalID(originalID)
		}
		if seller, ok := data["seller"].(string); ok {
			create.SetSeller(seller)
		}
		if rating, ok := data["average_rating"].(string); ok {
			create.SetAverageRating(rating)
			if ratingNum, err := strconv.ParseFloat(rating, 64); err == nil {
				create.SetRatingNumeric(ratingNum)
			}
		}
		if outOfStock, ok := data["out_of_stock"].(bool); ok {
			create.SetOutOfStock(outOfStock)
		}

		// Handle images array
		if images, ok := data["images"].([]interface{}); ok {
			imageStrings := make([]string, 0, len(images))
			for _, img := range images {
				if str, ok := img.(string); ok {
					imageStrings = append(imageStrings, str)
				}
			}
			if len(imageStrings) > 0 {
				create.SetImages(imageStrings)
			}
		}

		// Handle product details
		if details, ok := data["product_details"].([]interface{}); ok {
			productDetails := make([]map[string]string, 0, len(details))
			for _, detail := range details {
				if detailMap, ok := detail.(map[string]interface{}); ok {
					strMap := make(map[string]string)
					for k, v := range detailMap {
						if str, ok := v.(string); ok {
							strMap[k] = str
						}
					}
					productDetails = append(productDetails, strMap)
				}
			}
			if len(productDetails) > 0 {
				create.SetProductDetails(productDetails)
			}
		}

		// Handle crawled_at
		if crawledAt, ok := data["crawled_at"].(string); ok {
			if t := parseTime(crawledAt); !t.IsZero() {
				create.SetCrawledAt(t)
			}
		}

		// Set featured for every 20th product
		if (i+1)%20 == 0 {
			create.SetFeatured(true)
		}

		bulk = append(bulk, create)
	}

	// Save in batches
	batchSize := 100
	seeded := 0
	for i := 0; i < len(bulk); i += batchSize {
		end := i + batchSize
		if end > len(bulk) {
			end = len(bulk)
		}

		batch := bulk[i:end]
		_, err := client.Product.CreateBulk(batch...).Save(ctx)
		if err != nil {
			logHelper.Errorf("Failed to save batch %d-%d: %v", i, end, err)
			continue
		}

		seeded += len(batch)
		logHelper.Infof("Seeded batch %d-%d, total: %d", i, end, seeded)
	}

	logHelper.Infof("Successfully seeded %d products", seeded)
	return nil
}

// seedSampleData seeds with minimal sample data
func seedSampleData(ctx context.Context, client *ent.Client, logger log.Logger) error {
	logHelper := log.NewHelper(logger)
	logHelper.Info("Seeding sample data...")

	sampleProducts := []*ent.ProductCreate{
		client.Product.Create().
			SetTitle("Solid Men Multicolor Track Pants").
			SetBrand("York").
			SetCategory("Clothing and Accessories").
			SetSubCategory("Bottomwear").
			SetDescription("Yorker trackpants made from 100% rich combed cotton giving it a rich look. Designed for Comfort, Skin friendly fabric, itch-free waistband & great for all year round use Proudly made in India").
			SetActualPrice("2,999").
			SetSellingPrice("921").
			SetDiscount("69% off").
			SetPriceNumeric(921).
			SetPid("TKPFCZ9EA7H5FYZH").
			SetOriginalID("fa8e22d6-c0b6-5229-bb9e-ad52eda39a0a").
			SetSeller("Shyam Enterprises").
			SetAverageRating("3.9").
			SetRatingNumeric(3.9).
			SetImages([]string{
				"https://rukminim1.flixcart.com/image/128/128/jr3t5e80/track-pant/z/y/n/m-1005combo2-yorker-original-imafczg3xfh5qqd4.jpeg?q=70",
				"https://rukminim1.flixcart.com/image/128/128/jr58l8w0/track-pant/w/d/a/l-1005combo8-yorker-original-imafczg3pgtxgraq.jpeg?q=70",
			}).
			SetProductDetails([]map[string]string{
				{"Style Code": "1005COMBO2"},
				{"Closure": "Elastic"},
				{"Pockets": "Side Pockets"},
				{"Fabric": "Cotton Blend"},
				{"Pattern": "Solid"},
				{"Color": "Multicolor"},
			}).
			SetURL("https://www.flipkart.com/yorker-solid-men-multicolor-track-pants/p/itmd2c76aadce459?pid=TKPFCZ9EA7H5FYZH&lid=LSTTKPFCZ9EA7H5FYZHVYXWP0&marketplace=FLIPKART&srno=b_1_1&otracker=browse&fm=organic&iid=177a46eb-d053-4732-b3de-fcad6ff59cbd.TKPFCZ9EA7H5FYZH.SEARCH&ssid=utkd4t3gb40000001612415717799").
			SetStyleCode("1005COMBO2").
			SetCrawledAt(parseTime("02/10/2021, 20:11:51")).
			SetFeatured(true),

		client.Product.Create().
			SetTitle("Solid Men Blue Track Pants").
			SetBrand("York").
			SetCategory("Clothing and Accessories").
			SetSubCategory("Bottomwear").
			SetDescription("Yorker trackpants made from 100% rich combed cotton giving it a rich look. Designed for Comfort, Skin friendly fabric, itch-free waistband & great for all year round use Proudly made in India").
			SetActualPrice("1,499").
			SetSellingPrice("499").
			SetDiscount("66% off").
			SetPriceNumeric(499).
			SetPid("TKPFCZ9EJZV2UVRZ").
			SetOriginalID("893e6980-f2a0-531f-b056-34dd63fe912c").
			SetSeller("Shyam Enterprises").
			SetAverageRating("3.9").
			SetRatingNumeric(3.9).
			SetImages([]string{
				"https://rukminim1.flixcart.com/image/128/128/kfyasnk0/track-pant/g/5/y/s-19876-yorker-original-imafwamyzrwjynkf.jpeg?q=70",
				"https://rukminim1.flixcart.com/image/128/128/kfyasnk0/track-pant/g/5/y/s-19876-yorker-original-imafwamynyeuu5zq.jpeg?q=70",
			}).
			SetProductDetails([]map[string]string{
				{"Style Code": "1005BLUE"},
				{"Closure": "Drawstring, Elastic"},
				{"Pockets": "Side Pockets"},
				{"Fabric": "Cotton Blend"},
				{"Pattern": "Solid"},
				{"Color": "Blue"},
			}).
			SetURL("https://www.flipkart.com/yorker-solid-men-blue-track-pants/p/itmfczez7v6rzwer?pid=TKPFCZ9EJZV2UVRZ&lid=LSTTKPFCZ9EJZV2UVRZ9HEITU&marketplace=FLIPKART&srno=b_1_2&otracker=browse&fm=organic&iid=177a46eb-d053-4732-b3de-fcad6ff59cbd.TKPFCZ9EJZV2UVRZ.SEARCH&ssid=utkd4t3gb40000001612415717799").
			SetStyleCode("1005BLUE").
			SetCrawledAt(parseTime("02/10/2021, 20:11:52")),
	}

	_, err := client.Product.CreateBulk(sampleProducts...).Save(ctx)
	if err != nil {
		return err
	}

	logHelper.Info("Sample data seeded successfully")
	return nil
}

// Helper function to parse price
func parsePrice(priceStr string) int {
	if priceStr == "" {
		return 0
	}

	// Remove currency symbols and commas
	cleaned := strings.ReplaceAll(priceStr, "₹", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	if price, err := strconv.Atoi(cleaned); err == nil {
		return price
	}

	return 0
}

// Helper function to parse time
func parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}

	layouts := []string{
		"02/01/2006, 15:04:05",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, timeStr)
		if err == nil {
			return t
		}
	}

	return time.Now()
}
