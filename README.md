1. needs docker
2. you need to set up .env file in root directory with these variables:
TELEGRAM_TOKEN=  
DB_HOST=  
DB_PORT=  
DB_USER=  
DB_PASSWORD=  
DB_NAME=  
3. postgres user has to have superuser status to create custom type "anime_id_and_last_episode". Maybe it will be fixed later.
4. docker-compose up --build  
