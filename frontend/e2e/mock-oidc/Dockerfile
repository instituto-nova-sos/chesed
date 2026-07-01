# Test-only mock OIDC issuer image. Stands in for Keycloak in the E2E stack.
# Do NOT use in any production compose file.
FROM node:20-alpine
WORKDIR /app
COPY package.json ./
RUN npm install --omit=dev
COPY server.mjs ./
EXPOSE 8080
CMD ["node", "server.mjs"]
