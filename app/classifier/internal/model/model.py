import torch.nn as nn

class UltraLiteClassifier(nn.Module):
    def __init__(self, vocab_size=500, num_class=3, embed_dim=128):
        super().__init__()
        self.embedding = nn.EmbeddingBag(vocab_size, embed_dim, sparse=False, mode='mean')
        # Direct mapping from embedding to output classes
        self.fc = nn.Linear(embed_dim, num_class)
        
    def forward(self, text, offsets):
        embedded = self.embedding(text, offsets)
        return self.fc(embedded)