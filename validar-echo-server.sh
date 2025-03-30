SERVER_NAME="server"
SERVER_PORT=12345
HELLO_WORLD_MESSAGE="hello_world"
DOCKER_NETWORK="tp0_testing_net"

SERVER_RESPONSE=$(docker run --rm --network="$DOCKER_NETWORK" busybox:latest sh -c "echo '$HELLO_WORLD_MESSAGE' | nc $SERVER_NAME $SERVER_PORT")

echo $SERVER_RESPONSE
if [ "$SERVER_RESPONSE" == "$HELLO_WORLD_MESSAGE" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi
