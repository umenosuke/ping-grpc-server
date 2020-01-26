#!/bin/bash

cd $(dirname $0)/

cd ./build/aclDatabaseGrpc
./server
./clientWebServer

cd $(dirname $0)/

cd ./build/authentication
./server

cd $(dirname $0)/
