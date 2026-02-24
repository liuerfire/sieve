package engine

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.SourceTitle}} - Sieve</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 900px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f8f9fa;
        }
        header {
            text-align: center;
            margin-bottom: 40px;
            padding: 30px;
            background: white;
            border-radius: 12px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.05);
        }
        header h1 {
            margin: 0;
            color: #1a1a1a;
        }
        .report-meta {
            color: #666;
            font-size: 0.9em;
            margin-top: 10px;
        }
        .item {
            background: #fff;
            padding: 30px;
            margin-bottom: 30px;
            border-radius: 12px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.05);
            transition: transform 0.2s;
        }
        .item:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 12px rgba(0,0,0,0.08);
        }
        .stars {
            color: #f39c12;
            font-size: 1.3em;
            margin-bottom: 8px;
        }
        h2 {
            margin-top: 0;
            margin-bottom: 12px;
            color: #2c3e50;
            line-height: 1.3;
        }
        h2 a {
            color: #2c3e50;
            text-decoration: none;
        }
        h2 a:hover {
            color: #3498db;
        }
        .meta {
            font-size: 0.85em;
            color: #7f8c8d;
            margin-bottom: 20px;
            border-bottom: 1px solid #f1f1f1;
            padding-bottom: 10px;
        }
        .reason {
            font-size: 0.9em;
            font-style: italic;
            color: #666;
            margin-bottom: 15px;
            padding: 8px 12px;
            background: #f1f3f5;
            border-radius: 4px;
        }
        .description {
            color: #444;
            font-size: 0.95em;
        }
        .description p {
            margin-top: 0;
        }
        a {
            color: #3498db;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <header>
        <h1>{{.SourceTitle}}</h1>
        <div class="report-meta">
            <span>Generated via <a href="{{.SourceURL}}">{{.SourceName}}</a></span> | 
            <span>Total Items: {{.TotalItems}}</span> | 
            <span>Generated At: {{.GeneratedAt}}</span>
        </div>
    </header>
    {{range .Items}}
    <div class="item">
        <div class="stars">{{stars .InterestLevel}}</div>
        <h2><a href="{{.Link}}" target="_blank">{{.Title}}</a></h2>
        <div class="meta">
            <strong>Source:</strong> {{.Source}} | 
            <strong>Date:</strong> {{.PubDate}}
        </div>
        {{if .Reason}}
        <div class="reason">
            <strong>Classification Reason:</strong> {{.Reason}}
        </div>
        {{end}}
        <div class="description">
            {{.Description}}
        </div>
    </div>
    {{end}}
</body>
</html>
`
