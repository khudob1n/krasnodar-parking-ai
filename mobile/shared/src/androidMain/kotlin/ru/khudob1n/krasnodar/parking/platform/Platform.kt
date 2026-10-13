package ru.khudob1n.krasnodar.parking.platform

import android.annotation.SuppressLint
import android.content.Context

@SuppressLint("StaticFieldLeak") // контекст приложения, не экрана
object Platform {
    private lateinit var context: Context

    /** Вызывается из Application.onCreate. */
    fun init(context: Context) {
        this.context = context.applicationContext
    }

    val appContext: Context get() = context
}
