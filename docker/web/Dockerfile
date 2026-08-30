FROM node:22-alpine AS dependencies

WORKDIR /src
COPY web/package.json web/package-lock.json ./
RUN npm ci

FROM dependencies AS dev

COPY web/ ./
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

FROM dependencies AS build

COPY web/ ./
RUN npm run build

FROM nginx:1.28-alpine

COPY docker/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html

EXPOSE 80
