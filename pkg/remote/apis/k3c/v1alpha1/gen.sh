#!/bin/bash
VENDOR=${VENDOR:-${HOME}/src/cri/vendor}
protoc --gofast_out=plugins=grpc:. -I=${VENDOR}:. api.proto
