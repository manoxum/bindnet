FROM node:22-alpine AS build
# Prisma precisa do OpenSSL do sistema para detectar a engine correta -
# sem isso ("apk add openssl" ausente), o binario da engine Prisma falha
# ao carregar em imagens Alpine (musl) com erro obscuro de JSON invalido.
RUN apk add --no-cache openssl
WORKDIR /src
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build

FROM node:22-alpine
RUN apk add --no-cache openssl
WORKDIR /app
COPY --from=build /src/node_modules ./node_modules
COPY --from=build /src/dist ./dist
COPY --from=build /src/prisma ./prisma
COPY package.json ./
ENTRYPOINT ["node", "dist/index.js"]
