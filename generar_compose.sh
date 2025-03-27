if [ "$#" -ne 2 ]; then
    echo "Usage: $0 {output_file} {clients_amount}"
    exit 1
fi

OUTPUT_FILE=$1
CLIENTS_AMOUNT=$2

DATASET_ZIP="./.data/dataset.zip"
DATASET_DIR="./.data"

if [ -f "$DATASET_ZIP" ]; then
    echo "Extracting dataset..."
    unzip -o "$DATASET_ZIP" -d "$DATASET_DIR"
else
    echo "Dataset file '$DATASET_ZIP' not found"
    exit 1
fi

cat > "$OUTPUT_FILE" <<EOL
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL=DEBUG
      - MAX_CONNECTIONS=5
    volumes:
      - ./server/config.ini:/app/config.ini
    networks:
      - testing_net
EOL

for ((i=1; i<=CLIENTS_AMOUNT; i++)); do
    cat >> "$OUTPUT_FILE" <<EOL

  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=$i
      - CLI_LOG_LEVEL=DEBUG
      - NOMBRE=Santiago Lionel
      - APELLIDO=Lorca
      - DOCUMENTO=30904465
      - NACIMIENTO=1999-03-17
      - NUMERO=757$i
    volumes:
      - ./client/config.yaml:/app/config.yaml
      - ./.data/agency-$i.csv:/app/agency.csv
    networks:
      - testing_net
    depends_on:
      - server
EOL
done

cat >> "$OUTPUT_FILE" <<EOL

networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
EOL