CREATE TABLE IF NOT EXISTS "order_statuses" (
                                                id serial NOT NULL PRIMARY KEY,
                                                "name" varchar(50) NOT NULL,
    "is_final" boolean DEFAULT false
    );

INSERT INTO "order_statuses" (name, is_final) VALUES ('cool_order_created', false);
INSERT INTO "order_statuses" (name, is_final) VALUES ('sbu_varification_pending', false);
INSERT INTO "order_statuses" (name, is_final) VALUES ('confirmed_by_mayor', false);
INSERT INTO "order_statuses" (name, is_final) VALUES ('changed_my_mind', true);
INSERT INTO "order_statuses" (name, is_final) VALUES ('failed', true);
INSERT INTO "order_statuses" (name, is_final) VALUES ('chinazes', true);
INSERT INTO "order_statuses" (name, is_final) VALUES ('give_my_money_back', true);

CREATE TABLE IF NOT EXISTS "events" (
                                        "event_id" uuid NOT NULL PRIMARY KEY,
                                        "order_id" uuid NOT NULL,
                                        "user_id" uuid NOT NULL,
                                        "order_status_id" int NOT NULL,
                                        "updated_at" timestamp NOT NULL,
                                        "created_at" timestamp NOT NULL,

                                        FOREIGN KEY ("order_status_id") REFERENCES "order_statuses" ("id")
    );

CREATE INDEX "index_events_on_order_status_id" ON "events" ("order_status_id");
