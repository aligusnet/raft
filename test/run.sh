#!/usr/bin/env bash

log_base_dir="logs"

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


function create_log_dirs() {
    base_name=$1
    for i in $(eval echo {$2..$3}); do
        mkdir "./$log_base_dir/$base_name$i"
    done
}

function clean_logs() {
    rm -rf "./$log_base_dir"
    mkdir "./$log_base_dir"
}

function main() {
    clean_logs
    create_log_dirs "server" 0 4
    create_log_dirs "client" 0 9
    build_client && build_server && build_and_run_test_app
}

main