dc-up:
	@docker-compose -f ./docker-compose.yml up -d

dc-stop:
	@docker-compose -f ./docker-compose.yml stop