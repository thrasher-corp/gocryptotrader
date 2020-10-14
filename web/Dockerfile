FROM node:14-alpine as build
COPY package.json package-lock.json ./
RUN npm set progress=false \
 && npm config set depth 0 \
 && npm cache clean --force
RUN npm i \
 && mkdir /app \
 && cp -R ./node_modules /app
WORKDIR /app
COPY . .
RUN $(npm bin)/ng build --prod --build-optimizer

FROM nginx:1.19.2-alpine
RUN rm -rf /var/www/html/*
COPY nginx/default.conf /etc/nginx/conf.d/
COPY --from=build /app/dist /var/www/html
CMD ["nginx", "-g", "daemon off;"]
