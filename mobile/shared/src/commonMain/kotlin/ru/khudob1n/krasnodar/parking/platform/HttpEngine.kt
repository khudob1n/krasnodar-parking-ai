package ru.khudob1n.krasnodar.parking.platform

import io.ktor.client.HttpClient

/** HTTP-клиент платформы: OkHttp на Android, URLSession на iOS. */
expect fun createHttpClient(): HttpClient
