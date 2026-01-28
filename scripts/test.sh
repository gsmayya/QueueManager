#/bin/sh

# few tests to run 

docker compose exec -T db psql -U nodequeue -d nodequeue -c "\dt"

docker compose exec -T db psql -U nodequeue -d nodequeue -c "SELECT * FROM resources ORDER BY id;"

docker compose exec -T db psql -U nodequeue -d nodequeue -c "SELECT COUNT(*) AS nodes FROM nodes;"

docker compose exec -T db psql -U nodequeue -d nodequeue -c "SELECT action, COUNT(*) FROM node_logs GROUP BY action ORDER BY 2 DESC;"

# for dev environment
docker compose -f docker-compose.dev.yml exec -T db psql -U nodequeue -d nodequeue -c "\dt"