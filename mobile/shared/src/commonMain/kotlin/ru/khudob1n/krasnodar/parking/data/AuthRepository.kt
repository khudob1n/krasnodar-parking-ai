package ru.khudob1n.krasnodar.parking.data

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import ru.khudob1n.krasnodar.parking.platform.KeyValueStore

/** Действие требует входа, а пользователь не вошёл (или вход истёк). */
class NotLoggedInException : Exception("Нужно войти по номеру телефона")

private const val SESSION_KEY = "auth.session"

/**
 * Вход по коду из звонка и токены. Сессия хранится на устройстве, поэтому после перезапуска приложения
 * входить заново не нужно (refresh-токен живёт 30 дней).
 */
class AuthRepository(private val api: ApiClient) {
    private val state = MutableStateFlow(load())
    private val refreshLock = Mutex()

    /** Текущий вход; null - не вошёл. */
    val session: StateFlow<AuthSession?> = state.asStateFlow()

    suspend fun requestCode(phone: String): CodeResult = api.requestCode(phone)

    suspend fun verify(phone: String, code: String) {
        save(api.verify(phone, code))
    }

    suspend fun logout() {
        val current = state.value ?: return
        save(null)
        // Сервер недоступен - не страшно: на устройстве токенов уже нет.
        try {
            api.logout(current.refreshToken)
        } catch (e: CancellationException) {
            throw e
        } catch (_: Exception) {
        }
    }

    /**
     * Запрос от имени пользователя. Access-токен живёт недолго: на ответ 401 один раз обновляем
     * токены и повторяем. Refresh тоже не принят - выходим ([NotLoggedInException]).
     */
    suspend fun <T> withToken(block: suspend (accessToken: String) -> T): T {
        val used = state.value ?: throw NotLoggedInException()
        try {
            return block(used.accessToken)
        } catch (e: ApiException) {
            if (e.status != 401) throw e
        }
        val fresh = refreshLock.withLock {
            val current = state.value ?: throw NotLoggedInException()
            // Пока ждали замок, токены мог обновить параллельный запрос.
            if (current.accessToken != used.accessToken) return@withLock current
            try {
                api.refresh(current.refreshToken).also(::save)
            } catch (e: ApiException) {
                if (e.status == 401) {
                    save(null)
                    throw NotLoggedInException()
                }
                throw e
            }
        }
        return block(fresh.accessToken)
    }

    private fun save(session: AuthSession?) {
        state.value = session
        KeyValueStore.set(SESSION_KEY, session?.let { json.encodeToString(AuthSession.serializer(), it) })
    }

    private fun load(): AuthSession? = KeyValueStore.get(SESSION_KEY)?.let {
        runCatching { json.decodeFromString(AuthSession.serializer(), it) }.getOrNull()
    }
}
