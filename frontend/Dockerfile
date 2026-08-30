# syntax=docker/dockerfile:1
FROM node:22-alpine AS deps
WORKDIR /app

# Install only production dependencies with npm ci
COPY package.json package-lock.json ./
RUN npm ci --only=production

# Build stage
FROM node:22-alpine AS builder
WORKDIR /app

# Install all dependencies (including devDeps) for building
COPY package.json package-lock.json ./
RUN npm ci

# Build with NEXT_PUBLIC_API_MODE=live to ensure the image contains the live build
# (never the mock build with MSW). .env.local is excluded via .dockerignore,
# so this ARG takes precedence at build time.
# NEXT_PUBLIC_* is INLINED AT BUILD TIME — setting it at runtime is a no-op.
# Defaults to "live" so an image is never accidentally built in mock mode from
# a stray .env.local (which .dockerignore also excludes). The ENV must reference
# the ARG, not restate the literal, or `--build-arg` would be silently ignored.
ARG NEXT_PUBLIC_API_MODE=live
ENV NEXT_PUBLIC_API_MODE=${NEXT_PUBLIC_API_MODE}
ARG BACKEND_URL=http://backend:8080
ENV BACKEND_URL=$BACKEND_URL

# Copy source
COPY . .

# Build the Next.js app with standalone output
RUN npm run build

# Runner stage: minimal production image
FROM node:22-alpine AS runner
WORKDIR /app

# Create a non-root user for security
RUN addgroup -g 1001 -S nodejs
RUN adduser -S nextjs -u 1001

# Copy only what standalone build produces
# The standalone output is self-contained except for static assets and public files
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

USER nextjs

EXPOSE 3000

# Healthcheck to verify the app is running
# Targets /api/health — a route served by THIS app. Do not point this at
# /health: that path is rewritten by next.config.ts to ${BACKEND_URL}/health,
# so the frontend would report the BACKEND's health as its own and flap
# whenever the backend is down or still starting.
# `.get()` alone does not fail on a connection error, so bind an error handler
# and give the request an explicit timeout; otherwise a refused connection
# exits 0 and the container reports healthy while serving nothing.
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD node -e "const r=require('http').get('http://127.0.0.1:3000/api/health',s=>process.exit(s.statusCode===200?0:1));r.on('error',()=>process.exit(1));r.setTimeout(5000,()=>process.exit(1))"

CMD ["node", "server.js"]
