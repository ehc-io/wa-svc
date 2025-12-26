# WhatsApp Service (wa-svc) - Documentation Index

Complete technical documentation for the WhatsApp Service integration platform.

## Table of Contents

### 1. Overview & Architecture
- [**01-ARCHITECTURE.md**](01-ARCHITECTURE.md) - System architecture, design patterns, and technical overview
  - Executive Summary
  - System Architecture
  - Core Components
  - Data Flow & Integration Points
  - Design Decisions & Rationale

### 2. API Documentation
- [**02-API-REFERENCE.md**](02-API-REFERENCE.md) - Complete REST API documentation
  - Authentication Endpoints
  - Messaging Endpoints
  - Search & Query Endpoints
  - Contact & Group Management
  - Media Handling
  - Webhook Configuration

### 3. Database & Storage
- [**03-DATABASE-SCHEMA.md**](03-DATABASE-SCHEMA.md) - SQLite database design and schema
  - Table Structures
  - Indexes & Performance
  - Full-Text Search (FTS5)
  - Data Models
  - Migration Strategy

### 4. Components & Modules
- [**04-COMPONENTS.md**](04-COMPONENTS.md) - Detailed module documentation
  - Internal Packages
  - WhatsApp Client Wrapper
  - Storage Layer
  - Service Manager
  - API Server
  - Webhook System

### 5. Configuration
- [**05-CONFIGURATION.md**](05-CONFIGURATION.md) - Environment variables and settings
  - Server Configuration
  - Authentication Settings
  - Webhook Configuration
  - Sync Settings
  - Debug & Logging

### 6. Deployment
- [**06-DEPLOYMENT.md**](06-DEPLOYMENT.md) - Production deployment guide
  - Docker Deployment
  - Docker Compose
  - Environment Setup
  - Security Considerations
  - Monitoring & Health Checks
  - Troubleshooting

### 7. CLI Usage
- [**07-CLI-GUIDE.md**](07-CLI-GUIDE.md) - Command-line interface documentation
  - Installation
  - Authentication
  - Message Management
  - Contacts & Groups
  - Search Capabilities
  - Media Downloads

### 8. Development
- [**08-DEVELOPMENT.md**](08-DEVELOPMENT.md) - Development guide
  - Project Structure
  - Building from Source
  - Testing
  - Contributing Guidelines
  - Code Patterns

## Quick Start

For immediate setup, see:
1. [Configuration Guide](05-CONFIGURATION.md) - Set up your environment
2. [Deployment Guide](06-DEPLOYMENT.md) - Deploy with Docker
3. [API Reference](02-API-REFERENCE.md) - Start making API calls

## Support & Resources

- **GitHub Repository**: https://github.com/steipete/wacli
- **Issues & Bugs**: Report via GitHub Issues
- **License**: See [LICENSE](../LICENSE) file

## Documentation Version

- **Version**: 1.0
- **Last Updated**: December 2025
- **Go Version**: 1.24+
- **whatsmeow Version**: Latest (December 2024)
