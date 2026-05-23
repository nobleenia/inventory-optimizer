cat internal/web/api_records.go | grep -v HandleGetRecordsHistory > temp.go
mv temp.go internal/web/api_records.go
