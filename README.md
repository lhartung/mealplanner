Meal Planner
============

The meal planner is a web-based utility for planning daily meals and creating
organized shopping lists.  It features a simple machine learning algorithm that
learns to classify ingredients for easier shopping. It supports multi-user
authentication, but there is no public host at this time. Being fully open
source, feel free to modify the code as you see fit and self-host.

Build and Run
-------------

docker-compose build
docker-compose up -d

Server Setup
------------

sudo apt-get update
sudo apt-get install nginx certbot
sudo snap install docker

sudo docker-compose build
sudo docker-compose up -d

sudo certbot certonly -d example.com -d www.example.com

nginx Configuration
-------------------

```
server {
        listen 80;
        server_name example.com;

        root html;
        index index.html;

        location / {
                root /var/www/html;
                try_files $uri $uri/ =404;

                error_page 403 =301 https://$host$request_uri;
                error_page 404 =301 https://$host$request_uri;
        }
}

server {
        listen 443;
        server_name example.com;

        ssl on;
        ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
        ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

        ssl_session_timeout 5m;

        index index.html;

        location / {
                proxy_pass http://127.0.0.1:3000/;
                proxy_set_header Host $host;
                proxy_set_header X-Real-IP $remote_addr;
                proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
                proxy_redirect http:// $scheme://;
        }
}
```
