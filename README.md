1. needs docker
2. you need to set up .env file in root directory with these variables:
TELEGRAM_TOKEN=  
DB_HOST=  
DB_PORT=  
DB_USER=  
DB_PASSWORD=  
DB_NAME=  
4. docker-compose up --build  
5. docker-compose will call init.sh, if custom type anime_id_and_last_episode, table and user are still not created init.sh will create them.
