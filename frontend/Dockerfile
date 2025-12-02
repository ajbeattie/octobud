FROM node:20

WORKDIR /app

# Copy package files first for better caching
COPY package*.json ./

# Install dependencies (use install instead of ci for dev flexibility)
RUN npm install

# Copy the rest of the application
COPY . .

CMD ["npm", "run", "dev", "--", "--host", "--port", "5173"]

