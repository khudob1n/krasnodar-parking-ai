plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

android {
    namespace = "ru.khudob1n.krasnodar.parking"
    compileSdk = 37

    defaultConfig {
        applicationId = "ru.khudob1n.krasnodar.parking"
        minSdk = 26
        targetSdk = 37
        versionCode = 1
        versionName = "0.1.0"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    // Адрес бэкенда (backend/). Переопределяется -PapiBaseUrl=... при сборке. Debug по умолчанию -
    // локальный бэкенд через `adb reverse tcp:8080 tcp:8080`: loopback на телефоне доступен без
    // отдельного разрешения на локальную сеть. Прод-адреса пока нет - хостинг не выбран.
    val apiBaseUrl = providers.gradleProperty("apiBaseUrl").map { it.trimEnd('/') }
    buildTypes {
        debug {
            buildConfigField("String", "API_BASE_URL", "\"${apiBaseUrl.getOrElse("http://127.0.0.1:8080")}\"")
        }
        release {
            buildConfigField("String", "API_BASE_URL", "\"${apiBaseUrl.getOrElse("http://127.0.0.1:8080")}\"")
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }
}

dependencies {
    implementation(libs.androidx.core)
    implementation(libs.androidx.activity.compose)
    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.foundation)
    implementation(libs.compose.material3)
    // Весь интерфейс и логика - общие с iOS (модуль shared).
    implementation(project(":shared"))
}
