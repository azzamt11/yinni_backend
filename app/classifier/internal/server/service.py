import grpc
from api.classifier.v1.classifier_pb2 import ClassifyReply
from api.classifier.v1.classifier_pb2_grpc import ClassifierServicer

from internal.model.classifier import (
    TextPredictor,
    extract_ordinal,
    extract_payment,
)

_predictor = None


def get_predictor():
    global _predictor
    if _predictor is None:
        _predictor = TextPredictor()
    return _predictor


class ClassifierService(ClassifierServicer):
    def __init__(self):
        self.predictor = get_predictor()

    def Classify(self, request, context):
        try:
            predicted_class, confidence = self.predictor.predict(request.text)

            value = request.text

            if predicted_class == "Select_Option":
                extracted = extract_ordinal(request.text)
                if extracted is not None:
                    value = extracted

            elif predicted_class == "Make_Payment":
                value = extract_payment(request.text)

            return ClassifyReply(
                type=predicted_class.lower(),
                confidence=float(confidence),
                value=str(value),
            )

        except Exception as e:
            context.set_details(str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            return ClassifyReply(
                type="unknown",
                confidence=0.0,
                value="",
            )
