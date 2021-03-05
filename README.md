# Comet
A simple, powerful and modular Minecraft Docker container

## Deploy (Docker)
```
docker build -t comet:latest . && docker run -p 8080:8080 -e MC_EULA=true --name comet comet:latest
```
