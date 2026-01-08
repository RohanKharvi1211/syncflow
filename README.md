# SyncFlow - Multi-Tenant Data Synchronization Platform

A comprehensive data synchronization platform that enables multiple companies to sync their Salesforce data with various accounting systems (QuickBooks, Tally) with configurable field mappings.

## Features

### Multi-Tenant Architecture
- **Company Management**: Support for multiple companies with isolated data
- **Tenant Isolation**: Each company's data is completely separated
- **Company Switching**: Easy switching between different companies in the UI

### Supported Integrations
- **Salesforce**: Source system for CRM data
- **QuickBooks**: Accounting system integration
- **Tally**: Accounting system integration
- **Extensible**: Easy to add new target systems

### Field Mapping System
- **Visual Field Mapping**: Configure how Salesforce fields map to accounting system fields
- **Multiple Mapping Types**:
  - Direct mapping (1:1 field mapping)
  - Transform mapping (with custom transformation rules)
  - Constant mapping (fixed values)
- **Transformation Rules**: Support for concatenation, splitting, formatting, and lookup transformations
- **Required Field Validation**: Mark fields as required for successful sync

### Real-time Synchronization
- **Webhook Support**: Real-time updates from Salesforce
- **Background Sync**: Scheduled synchronization processes
- **Retry Mechanism**: Automatic retry for failed syncs
- **Status Tracking**: Complete visibility into sync status

### Dashboard & Monitoring
- **Real-time Dashboard**: Live statistics and monitoring
- **Sync Records**: Detailed view of all synchronization attempts
- **Error Logging**: Comprehensive error tracking and logging
- **Activity Feed**: Recent sync activities and status updates

## Architecture

### Backend (Go)
```
backend/
├── models/           # Database models
├── services/         # Business logic services
├── handlers/         # HTTP request handlers
├── config/           # Configuration management
├── middleware/       # HTTP middleware
└── main.go          # Application entry point
```

### Frontend (HTML/CSS/JavaScript)
```
frontend/
├── css/             # Stylesheets
├── js/              # JavaScript application
└── index.html       # Main application page
```

## Database Schema

### Core Tables
- **companies**: Company information and tenant isolation
- **integrations**: Integration configurations per company
- **field_mappings**: Field mapping configurations
- **sync_records**: Individual sync record tracking
- **sync_logs**: Detailed sync operation logs

### Key Features
- **Soft Deletes**: Records are soft-deleted for audit trails
- **Audit Fields**: Created/updated timestamps on all records
- **Foreign Key Constraints**: Proper relational integrity

## Installation & Setup

### Prerequisites
- Go 1.19 or higher
- PostgreSQL 12 or higher
- Node.js (optional, for frontend development)

### Backend Setup

1. **Clone and navigate to backend directory**:
   ```bash
   cd backend
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Set up environment variables**:
   ```bash
   cp .env.example .env
   # Edit .env with your database and API credentials
   ```

4. **Run database migrations**:
   ```bash
   go run main.go
   # The application will automatically migrate the database schema
   ```

5. **Start the server**:
   ```bash
   go run main.go
   ```

### Frontend Setup

1. **Navigate to frontend directory**:
   ```bash
   cd frontend
   ```

2. **Open in browser**:
   ```bash
   # Simply open index.html in your browser
   # Or serve with a local server:
   python -m http.server 8000
   ```

## Configuration

### Environment Variables

```env
# Database Configuration
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=syncflow
DB_PORT=5432

# Server Configuration
PORT=8080

# Salesforce Configuration (per company)
SALESFORCE_CLIENT_ID=your_client_id
SALESFORCE_CLIENT_SECRET=your_client_secret
SALESFORCE_USERNAME=your_username
SALESFORCE_PASSWORD=your_password
SALESFORCE_SECURITY_TOKEN=your_security_token
SALESFORCE_INSTANCE_URL=https://your-instance.salesforce.com

# QuickBooks Configuration (per company)
QUICKBOOKS_CLIENT_ID=your_client_id
QUICKBOOKS_CLIENT_SECRET=your_client_secret
QUICKBOOKS_ACCESS_TOKEN=your_access_token
QUICKBOOKS_REFRESH_TOKEN=your_refresh_token
QUICKBOOKS_REALM_ID=your_realm_id

# Tally Configuration (per company)
TALLY_SERVER_URL=your_tally_server
TALLY_PORT=9000
TALLY_USERNAME=your_username
TALLY_PASSWORD=your_password
```

## Usage

### 1. Company Setup

1. **Create a Company**:
   - Navigate to the Companies tab
   - Click "Add Company"
   - Fill in company name and domain
   - Save the company

2. **Select Company**:
   - Use the company selector dropdown
   - All subsequent operations will be scoped to the selected company

### 2. Integration Configuration

1. **Add Salesforce Integration**:
   - Go to Integrations tab
   - Click "Add Integration"
   - Select "Salesforce" as type
   - Provide Salesforce credentials in JSON format

2. **Add Target System Integration**:
   - Add QuickBooks or Tally integration
   - Configure with appropriate credentials

### 3. Field Mapping Setup

1. **Navigate to Field Mappings**:
   - Select source system (Salesforce)
   - Select target system (QuickBooks/Tally)
   - Click "Load Mappings"

2. **Create Field Mappings**:
   - Click "Add Mapping"
   - Configure source and target fields
   - Set mapping type and transformation rules
   - Mark required fields as needed

### 4. Synchronization

1. **Manual Sync**:
   - Go to Dashboard
   - Click sync buttons for specific record types
   - Monitor progress in real-time

2. **Monitor Sync Status**:
   - View sync records in the Records tab
   - Check dashboard statistics
   - Review error logs for failed syncs

## API Endpoints

### Company Management
- `GET /api/companies` - List all companies
- `POST /api/companies` - Create new company
- `GET /api/companies/{id}` - Get company details
- `PUT /api/companies/{id}` - Update company
- `DELETE /api/companies/{id}` - Delete company

### Field Mapping Management
- `GET /api/field-mappings` - Get field mappings
- `POST /api/field-mappings` - Create field mapping
- `PUT /api/field-mappings/{id}` - Update field mapping
- `DELETE /api/field-mappings/{id}` - Delete field mapping
- `POST /api/field-mappings/test` - Test field mapping

### Synchronization
- `POST /api/sync/{company_id}/{record_type}` - Trigger sync
- `GET /api/sync/records` - Get sync records
- `GET /api/sync/stats` - Get sync statistics
- `POST /api/sync/retry` - Retry failed syncs

## Field Mapping Examples

### Direct Mapping
```json
{
  "source_field": "Name",
  "target_field": "CompanyName",
  "mapping_type": "direct"
}
```

### Transform Mapping (Concatenation)
```json
{
  "source_field": "FirstName",
  "target_field": "DisplayName",
  "mapping_type": "transform",
  "transform_rule": {
    "type": "concat",
    "fields": ["source", " ", "LastName"],
    "separator": " "
  }
}
```

### Lookup Mapping
```json
{
  "source_field": "StageName",
  "target_field": "Status",
  "mapping_type": "transform",
  "transform_rule": {
    "type": "lookup",
    "lookup": {
      "Closed Won": "Completed",
      "Closed Lost": "Cancelled",
      "Prospecting": "In Progress"
    },
    "default": "Unknown"
  }
}
```

## Development

### Adding New Target Systems

1. **Create Model**: Add new model in `models/` directory
2. **Create Service**: Implement service in `services/` directory
3. **Update Sync Service**: Add integration to `MultiTenantSyncService`
4. **Update UI**: Add new system to frontend dropdowns

### Adding New Field Transformations

1. **Extend FieldMappingService**: Add new transformation type
2. **Update UI**: Add transformation options to frontend
3. **Add Validation**: Ensure proper validation for new transformation rules

## Security Considerations

- **API Authentication**: Implement proper authentication for production use
- **Data Encryption**: Encrypt sensitive configuration data
- **Access Control**: Implement role-based access control
- **Audit Logging**: Comprehensive audit trails for all operations

## Monitoring & Logging

- **Structured Logging**: All operations are logged with structured data
- **Error Tracking**: Comprehensive error tracking and reporting
- **Performance Metrics**: Monitor sync performance and success rates
- **Health Checks**: Built-in health check endpoints

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue in the repository
- Check the documentation
- Review the API endpoints

## Roadmap

- [ ] Real-time WebSocket updates
- [ ] Advanced transformation rules
- [ ] Bulk sync operations
- [ ] Data validation rules
- [ ] Custom webhook endpoints
- [ ] Advanced reporting and analytics
- [ ] Mobile application
- [ ] API rate limiting
- [ ] Advanced security features
