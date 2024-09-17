FROM ubuntu:24.04

# Install Java (JRE only) and other necessary dependencies
RUN apt-get update && \
    apt-get install --no-install-recommends -y openjdk-21-jre-headless && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Copy the JAR file and necessary libraries
COPY build/libs/ libs/

# placehoolder: expose ports
# EXPOSE <port>

# placehoolder: entrypoint
# ENTRYPOINT ["paladin"]

# Run the JAR file
# Override this with `docker run --entrypoint`
ENTRYPOINT [                         \
    "java",                          \ 
    "-Djna.library.path=/app/libs",  \
    "-jar",                          \
    "/app/libs/paladin.jar"           \
    ]

# Default command-line arguments (config file and node name)
# Override this with docker run <image> /app/other-config.yaml another-node

CMD ["/app/config.paladin.yaml", "testbed"]

# java -Djna.library.path=/app/libs -jar /app/libs/paladin.jar /app/config.paladin.yaml testbed
# java -Djna.library.path=core/go/build/libs -Djna.library.path=toolkit/go/build/libs -jar core/java/build/libs/paladin.jar core/go/test/config/postgres.config.yaml testbed

# java -Djna.library.path=build/libs -jar build/libs/paladin.jar core/go/test/config/postgres.config.yaml testbed

# Override the default command-line arguments
# docker run -d -p <>:<> \
# docker run -d --name=paladon \
#   -v ./core/go/db/migrations:/app/db/migrations \
#   -v ./core/go/test/config/postgres.config.yaml:/app/config.paladin.yaml \
#   paladon
#   <image> \
#   -Djna.library.path=/app \
#   -jar paladin.jar <path to config file> <node name>


# docker buildx build --platform linux/arm64 -t paladin .

# docker run -it --entrypoint=bash \
#   -v ./core/go/db/migrations:/app/db/migrations \
#   -v ./core/go/test/config/postgres.config.yaml:/app/config.paladin.yaml \
#   bla/paladin