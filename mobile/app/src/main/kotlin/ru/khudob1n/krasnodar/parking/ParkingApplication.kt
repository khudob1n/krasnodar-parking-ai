package ru.khudob1n.krasnodar.parking

import android.app.Application
import ru.khudob1n.krasnodar.parking.platform.AppConfig
import ru.khudob1n.krasnodar.parking.platform.Platform

class ParkingApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        Platform.init(this)
        AppConfig.apiBaseUrl = BuildConfig.API_BASE_URL
        AppConfig.versionName = BuildConfig.VERSION_NAME
        AppConfig.isDebug = BuildConfig.DEBUG
    }
}
