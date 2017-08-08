# golang-test-task

Сервис по поиску отелей в Абзаково

## Сборка и запуск

1. Установить [gb](https://getgb.io/)
1. Восстановить зависимости: `gb vendor restore`
1. Собрать проект: `gb build`
1. Запустить elasticsearch (`docker-compose up -d db`)
1. Запуск: `./bin/server`


## Описание методов

1. Поиск по отелям:

    * Поиск по названию: 

        ``` curl -d '{"name": "Spa"}' localhost:8081/search | jq . ```

    * Поиск по координатам:

        ``` curl -d '{"location": {"lat": 53.8077280023929, "lon": 58.6349773406982}, "radius": "150m"}' localhost:8081/search | jq . ```

    * Поиск по адресу: 

        ``` curl -d '{"address": "Горнолыжная"}' localhost:8081/search | jq . ```

    * В каждый из запросов можно указать дополнительный параметр ```language``` для фильтрации по коду языка ("en", "ru")

2. Получение отеля по ID (указывается язык получаемого документа, по умолчанию -- английский):

    ``` curl -d '{"id": "gorniy-vozdukh", "language": "en"}' localhost:8081/get | jq .```


