
run() {
    ${DOCKER} compose up    
}

stop() {
    ${DOCKER} compose down    
}

build() {
    ${DOCKER} build --platform "${OS}/${ARCH}" -t "user-service:latest" -f ./service/user/Dockerfile .
    ${DOCKER} build --platform "${OS}/${ARCH}" -t "event-service:latest" -f ./service/event/Dockerfile .
    ${DOCKER} build --platform "${OS}/${ARCH}" -t "event-log-service:latest" -f ./service/event_log/Dockerfile .
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

ARCH=$(uname -m)
OS=linux

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

