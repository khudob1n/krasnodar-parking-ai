# Мобильное приложение

Kotlin Multiplatform + Compose Multiplatform: весь интерфейс и логика в модуле `shared`
(Android и iOS), `app` - оболочка Android. Тема, шрифты Onest и JetBrains Mono и форма
«сквиркл» взяты из дизайн-системы приложения «Транспорт Краснодара».

## Запуск на Android

1. Поднять бэкенд (`backend/README.md`), он слушает `:8080`.
2. Подключить телефон и пробросить порт: `adb reverse tcp:8080 tcp:8080`.
3. `cd mobile && ./gradlew :app:installDebug`.

Другой адрес бэкенда: `./gradlew :app:installDebug -PapiBaseUrl=https://...`.

Готовый debug-APK собирает GitHub Actions (workflow `mobile`, артефакт `krasnodar-parking-debug-apk`).
Если в репозитории задана переменная `API_BASE_URL` (адрес сервера, см. `deploy/FREE.md` в ветке бэкенда),
этот APK ходит на сервер; иначе - на `127.0.0.1:8080`.
