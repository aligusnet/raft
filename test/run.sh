#!/usr/bin/env bash

function build_client() {
    echo 'building client...'
    cd client
    pwd
    rm -f client
    go build
    cd ..
}

function build_server() {
    echo 'building server...'
    cd server
    pwd
    rm -f server
    go build
    cd ..
}

function build_and_run_test_app() {
    echo 'building and running testapp...'
    pwd
    rm -f testapp
    go build && ./test -logtostderr=true
}


make -C logs
build_client && build_server && build_and_run_test_app
