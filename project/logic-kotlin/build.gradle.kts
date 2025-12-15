plugins {
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.spring)
    alias(libs.plugins.spring.boot)
    alias(libs.plugins.spring.dependency.management)
    alias(libs.plugins.protobuf)
}

group = "com.sudooom.mahjong"
version = "1.0.0"

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
    }
}

kotlin {
    compilerOptions {
        freeCompilerArgs.addAll("-Xjsr305=strict")
    }
}

repositories {
    mavenCentral()
}

dependencies {
    // Spring Boot Core - 无 Web 依赖
    implementation(libs.spring.boot.starter)

    // R2DBC - 响应式数据库
    implementation(libs.spring.boot.starter.data.r2dbc)
    implementation(libs.r2dbc.postgresql)

    // Reactive Redis
    implementation(libs.spring.boot.starter.data.redis.reactive)

    // Kotlin Coroutines
    implementation(libs.bundles.coroutines)

    // NATS
    implementation(libs.jnats)

    // Protobuf (用于消息序列化)
    implementation(libs.protobuf.kotlin)

    // Jackson for JSON
    implementation(libs.jackson.module.kotlin)

    // Test
    testImplementation(libs.bundles.testing)
}

protobuf {
    protoc {
        artifact = "com.google.protobuf:protoc:${libs.versions.protobuf.get()}"
    }
    generateProtoTasks {
        all().forEach {
            it.builtins {
                create("kotlin")
            }
        }
    }
}

tasks.withType<Test> {
    useJUnitPlatform()
}
