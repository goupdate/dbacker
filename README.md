# dbacker - PostgreSQL Auto-Backup Tool
PostgreSQL (especially for DigitalOcean) automatically SQL database backupper. Store backupped copy of tables in database.
A lightweight Go application that performs automated daily backups of PostgreSQL tables and manages backup retention.

## Features

- **Automated daily backups** of all tables in the specified database
- **Configurable retention policy** (default: 14 days)
- **Simple naming convention**: `autobackup_originaltable_YYYYMMDD`
- **Automatic cleanup** of old backups
- **Easy configuration** via JSON config file

## Installation

1. Ensure you have Go installed (version 1.16+ recommended)
2. Clone this repository:
   ```bash
   git clone https://github.com/goupdate/dbacker.git
   cd dbacker
   ```
3. Install dependencies:
   ```bash
   go get github.com/lib/pq
   ```

## Configuration

Create a `config.ini` file in the project directory with your PostgreSQL connection details:

```json
{
	"postgres": {
		"host": "localhost",
		"port": 123,
		"user": "wer",
		"password": "password",
		"dbname": "dbname"
	},
	"backup": {
		"prefix": "autobackup",
		"retention": 14
	}
}
```

### Configuration Options

| Section   | Option     | Description                                                                 | Default     |
|-----------|------------|-----------------------------------------------------------------------------|-------------|
| postgres  | host       | PostgreSQL server hostname                                                  | localhost   |
|           | port       | PostgreSQL server port                                                      | 5432        |
|           | user       | Database username                                                           | -           |
|           | password   | Database password                                                           | -           |
|           | dbname     | Database name to backup                                                     | -           |
| backup    | prefix     | Prefix for backup tables (e.g., "autobackup")                               | autobackup  |
|           | retention  | Number of days to keep backups (older backups will be deleted automatically)| 14          |

## Usage

### Manual Run

Build and run the application:

```bash
go build -o dbacker
./dbacker
```

### Scheduled Execution (Linux)

Add to crontab for daily execution at 2 AM:

```bash
0 2 * * * /path/to/dbacker >> /var/log/dbacker.log 2>&1
```

## Backup Strategy

The application implements the following backup logic:

1. Connects to the specified PostgreSQL database
2. Deletes all backup tables older than the configured retention period (14 days by default)
3. Creates new backups of all non-backup tables using the pattern: `{prefix}_{original_table}_{date}`
   - Example: `autobackup_users_20230501` for the `users` table backed up on May 1, 2023

