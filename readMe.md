Preinstall:
1. Go
2. Postgres
3. Docker

To start write in the terminal:
`make dc-up`

POST a message to db:
`curl --location 'http://localhost:8080/webhooks/payments/orders' \
--header 'Content-Type: application/json' \
--data '{
"event_id":"5999cfb6-a335-409a-879c-68b3b59aa660",
"order_id":"0fffb19b-b9d0-44a4-8bc2-480a78456033",
"user_id":"48a388a3-c388-47a5-b023-c1e61b70eae6",
"order_status":"cool_order_created",
"updated_at":"2019-01-01T00:00:00Z",
"created_at":"2019-01-01T00:00:00Z"
}'`

Allowed order statuses:
`cool_order_created,
sbu_varification_pending,
confirmed_by_mayor,
changed_my_mind,
failed,
chinazes,
give_my_money_back
`

Connect to the stream:
`curl --location 'http://localhost:8080/orders/<ORDER_ID>/events'`

Allowed to GET info about orders:
`curl --location 'http://localhost:8080/orders?user_id=48a388a3-c388-47a5-b023-c1e61b70eae6&is_final=false&limit=1&offset=1'`

allowed query parameters:
mandatory parameter one of the following:
`is_final` - `true|false`;
`status` - `[cool_order_created,sbu_varification_pending,confirmed_by_mayor,changed_my_mind,failed,chinazes,give_my_money_bac]`;
optional:
`user_id` - `uuid`;
`limit`;
`offset`;
