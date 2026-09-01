FROM mcr.microsoft.com/playwright:v1.55.0-noble
WORKDIR /test
COPY package.json ./
RUN npm install --omit=dev
COPY e2e.mjs ./
ENTRYPOINT ["node", "e2e.mjs"]
