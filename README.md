**jlink.online** is a HTTP microservice that builds optimized/minimized Java runtimes on the fly.

## Introduction
As of Java 9, the JDK now includes a tool called `jlink` that can build optimized runtimes for modular Java applications. Using an optimized runtime is a good idea when deploying to a production environment or when bundling a platform-specific runtime to distribute with your application.

This project is a wrapper for Java's `jlink` utility that makes it faster and easier to build custom Java runtimes. Just send it a request containing the names of the modules you need and **jlink.online** will automatically fetch the appropriate JDK, run `jlink` to produce a minimal runtime, then return that compressed runtime in the response body.

## Usage Examples
#### Download Java 11 for Linux x64 (containing `java.base` only)
```
https://jlink.online/x64/linux/11.0.8+10
```

#### Download Java 11 for Linux x64 (containing `java.desktop` and `jdk.zipfs`)
```
https://jlink.online/x64/linux/11.0.8+10?modules=java.desktop,jdk.zipfs
```

#### Download a runtime in a Dockerfile
```sh
# If you do 'FROM openjdk' then you'll get a full runtime
FROM alpine:latest

# Install dependencies
RUN apk add curl

# Install custom runtime
RUN curl -G 'https://jlink.online/x64/linux/lts' \
    -d modules=java.base \
    | tar zxf -

# Install application
# ...
```

#### Upload your application's `module-info.java` (experimental)
Suppose your application has the following module definition:
```java
module com.github.example {
    requires org.slf4j;
}
```

Then to build a custom runtime containing your dependencies, you can send a POST request containing the contents of your `module-info.java`:
```sh
curl --data-binary @com.github.example/src/main/java/module-info.java \
  'https://jlink.online/x64/linux/lts?artifacts=org.slf4j:slf4j-api:2.0.0-alpha1' \
  --output app_runtime.tar.gz
```

The `artifacts` parameter is a comma-separated list of the Maven Central coordinates of your dependencies. This is required to know what versions to include in your runtime.

**Unfortunately this can't work for dependencies that are automatic modules (because automatic modules don't specify *their* dependencies).**

## Credits
Thanks to the following projects:

- [gin](https://github.com/gin-gonic/gin) - for handling HTTP requests
- [jlink](https://docs.oracle.com/javase/9/tools/jlink.htm) - for minimizing runtime images
- [testify](https://github.com/stretchr/testify) - for unit testing
