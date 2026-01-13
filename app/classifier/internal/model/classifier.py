import torch
import json
import pickle
import os
import sys
from internal.model.model import UltraLiteClassifier
from rapidfuzz import process, fuzz

# 1. Get the directory of THIS file (/app/internal/model/)
CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))

# 2. Go up 2 levels to reach /app/
# (internal -> model -> /app/)
BASE_DIR = os.path.abspath(os.path.join(CURRENT_DIR, "../.."))

model_path = os.path.join(BASE_DIR, "ulc_model_weights.pth")
vocab_path = os.path.join(BASE_DIR, "ulc_vocab.pkl")
config_path = os.path.join(BASE_DIR, "ulc_model_config.json")

class TextPredictor:
    def __init__(self, model_path=model_path, 
                 vocab_path=vocab_path, 
                 config_path=config_path):
        """
        Load trained model for inference
        """
        # Check if files exist
        missing_files = []
        for path in [model_path, vocab_path, config_path]:
            if not os.path.exists(path):
                missing_files.append(path)
        
        if missing_files:
            for f in missing_files:
                print(f"   - {f}")
            raise FileNotFoundError(f"Missing files: {missing_files}")
        
        # Load configuration
        with open(config_path, 'r') as f:
            self.config = json.load(f)
        
        # Load vocabulary
        with open(vocab_path, 'rb') as f:
            self.vocab = pickle.load(f)
        
        # Get ACTUAL vocabulary size from the loaded vocab
        actual_vocab_size = len(self.vocab)
        config_vocab_size = self.config.get('vocab_size', actual_vocab_size)
        
        # Use the larger of the two to be safe
        vocab_size_to_use = max(actual_vocab_size, config_vocab_size)
        
        # Initialize model with CORRECT vocabulary size
        self.model = UltraLiteClassifier(
            vocab_size=vocab_size_to_use,
            num_class=self.config['num_classes'],
            embed_dim=self.config.get('embed_dim', 128)
        )
        
        # Load trained weights
        checkpoint = torch.load(model_path, map_location="cpu")

        if isinstance(checkpoint, dict) and "state_dict" in checkpoint:
            checkpoint = checkpoint["state_dict"]

        self.model.load_state_dict(checkpoint, strict=False)
        
        try:
            self.model.load_state_dict(checkpoint, strict=True)
        except RuntimeError as e:
            # Try partial load
            self.model.load_state_dict(checkpoint, strict=False)
        
        self.model.eval()  # Set to evaluation mode
        
        # Get class names
        self.class_names = self.config.get('class_names', ['Type_0', 'Type_1', 'Type_2'])
    
    def tokenize_text(self, text):
        """
        Convert text to token indices using the saved vocabulary
        """
        # Same preprocessing as during training
        text = text.lower().strip()
        words = text.split()
        
        tokens = []
        for word in words:
            # Use vocab.get(word, 0) where 0 is <unk> token
            token = self.vocab.get(word, 0)
            if token >= self.model.embedding.num_embeddings:
                token = 0  # Fallback to <unk> if token ID is out of bounds
            tokens.append(token)
        
        if not tokens:  # If empty input
            tokens = [0]  # Use unknown token
        
        return torch.tensor(tokens, dtype=torch.long)
    
    def predict(self, text, return_details=False):
        """
        Predict class for input text
        """
        try:
            # Tokenize
            tokens = self.tokenize_text(text)
            
            # Check if we have any valid tokens
            if len(tokens) == 0:
                return "Unknown" if not return_details else {"error": "No valid tokens"}
            
            # Prepare for EmbeddingBag
            text_tensor = tokens
            offsets = torch.tensor([0])
            
            # Inference
            with torch.no_grad():
                output = self.model(text_tensor, offsets)
                probabilities = torch.softmax(output, dim=1)
                predicted_class_idx = torch.argmax(probabilities, dim=1).item()
            
            predicted_class = self.class_names[predicted_class_idx]
            confidence = probabilities[0][predicted_class_idx].item()
            
            if return_details:
                # Get all probabilities
                probs_dict = {}
                for i, class_name in enumerate(self.class_names):
                    probs_dict[class_name] = probabilities[0][i].item()
                
                return {
                    'text': text,
                    'predicted_class': predicted_class,
                    'class_id': predicted_class_idx,
                    'confidence': confidence,
                    'probabilities': probs_dict,
                    'num_tokens': len(tokens)
                }
            else:
                return predicted_class, confidence
                
        except Exception as e:
            print(f"Prediction error: {e}")
            if return_details:
                return {"error": str(e)}
            return "Error", 0.0
    
    def predict_batch(self, texts):
        """
        Predict classes for multiple texts at once
        """
        results = []
        for text in texts:
            try:
                pred_class, confidence = self.predict(text)
                results.append({
                    'text': text,
                    'predicted_class': pred_class,
                    'confidence': confidence
                })
            except Exception as e:
                results.append({
                    'text': text,
                    'error': str(e),
                    'predicted_class': 'Error',
                    'confidence': 0.0
                })
        
        return results

ORDINAL_MAP = {
    "first": 1,
    "second": 2,
    "third": 3,
    "fourth": 4,
    "fifth": 5,
    "sixth": 6,
    "seventh": 7,
    "eighth": 8,
    "ninth": 9,
    "tenth": 10,
    "last": -1,
    "1": 1,
    "2": 2,
    "3": 3,
    "4": 4,
    "5": 5,
    "6": 6,
    "7": 7,
    "8": 8,
    "9": 9,
    "10": 10,
}

PAYMENT_MAP = {
    "ovo": "OVO",
    "dana": "DANA",
    "gopay": "GOPAY",
    "shopeepay": "SHOPEEPAY",
    "bca": "BCA",
    "bri": "BRI",
    "mandiri": "MANDIRI",
    "bni": "BNI",
    "mastercard": "MASTERCARD",
    "visa": "VISA",
    "kredivo": "KREDIVO",
}

def extract_ordinal(text, threshold=60):
    tokens = text.lower().split()

    match, score, _ = process.extractOne(
        " ".join(tokens),
        ORDINAL_MAP.keys(),
        scorer=fuzz.partial_ratio
    )

    if score >= threshold:
        return ORDINAL_MAP[match]

    return None

def extract_payment(text, threshold=70):
    tokens = text.lower().split()

    match, score, _ = process.extractOne(
        " ".join(tokens),
        PAYMENT_MAP.keys(),
        scorer=fuzz.partial_ratio
    )

    if score >= threshold:
        return PAYMENT_MAP[match]

    return "UNSPECIFIED"