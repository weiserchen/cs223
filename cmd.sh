
run() {
    ${DOCKER} compose up    
}

stop() {
    ${DOCKER} compose down    
}

build() {
    ${DOCKER} build -f ./service/user/Dockerfile .
    # ${DOCKER} build -f ./service/event/Dockerfile .
    # ${DOCKER} build -f ./service/event_log/Dockerfile .
}

cmd=$1

if docker version &> /dev/null; then
    DOCKER="docker"
elif podman version &> /dev/null; then
    DOCKER="podman"
else
    echo "unsupported container runtime"
    exit 1
fi

if [[ "${cmd}" == "run" ]]; then 
    trap stop exit
    run
elif [[ "${cmd}" == "stop" ]]; then
    stop
elif [[ "${cmd}" == "build" ]]; then
    build
else
    echo "unsupported command: ${cmd}"
    exit 1
fi

