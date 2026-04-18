FROM node:24.14.1-alpine3.22

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

ARG API_PROXY_TARGET=http://app:8080
ENV API_PROXY_TARGET=${API_PROXY_TARGET}
ENV NEXT_TELEMETRY_DISABLED=1

RUN npm run build

EXPOSE 3000

CMD ["npm", "run", "start", "--", "-p", "3000"]
