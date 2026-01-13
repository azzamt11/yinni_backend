import grpc
from concurrent import futures
from api.classifier.v1.classifier_pb2_grpc import (
    add_ClassifierServicer_to_server,
)
from internal.server.service import ClassifierService

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    add_ClassifierServicer_to_server(ClassifierService(), server)
    server.add_insecure_port("[::]:9005")
    server.start()
    server.wait_for_termination()

if __name__ == "__main__":
    serve()
