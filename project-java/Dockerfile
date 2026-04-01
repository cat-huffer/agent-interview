# 多阶段构建 - 构建阶段
FROM maven:3.9-eclipse-temurin-17 AS builder
WORKDIR /build
COPY pom.xml .
RUN mvn dependency:go-offline -B
COPY src ./src
RUN mvn clean package -DskipTests -B

# 运行阶段
FROM eclipse-temurin:17-jre-alpine
LABEL maintainer="agent-platform"

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /build/target/agent-platform-*.jar app.jar

RUN chown -R app:app /app
USER app

EXPOSE 8080

ENV JAVA_OPTS="-Xms512m -Xmx1024m -XX:+UseG1GC -XX:MaxGCPauseMillis=200"

ENTRYPOINT ["sh", "-c", "java ${JAVA_OPTS} -jar app.jar"]
