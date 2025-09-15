# Node.js server Dockerfile
FROM node:18-alpine

# Create app user for security
RUN addgroup -g 1001 -S nodejs
RUN adduser -S nodejs -u 1001

# Set working directory
WORKDIR /app

COPY package*.json ./
# npm clean install
RUN npm ci --only=production && npm cache clean --force

# Copy server source code
COPY --chown=nodejs:nodejs app.js ./
COPY --chown=nodejs:nodejs bin/ ./bin/
COPY --chown=nodejs:nodejs routes/ ./routes/
COPY --chown=nodejs:nodejs public/ ./public/
COPY --chown=nodejs:nodejs data/ ./data/

# Create necessary directories and set permissions
RUN mkdir -p logs && chown -R nodejs:nodejs /app && chmod -R 755 /app/data

# Switch to non-root user
USER nodejs

EXPOSE 3001

CMD ["npm", "start"]
