package engine

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.SourceTitle}} - Sieve</title>
    <style>
        :root {
            --high-bg: #fffdf5;
            --high-border: #f39c12;
            --interest-bg: #fff;
            --uninterested-bg: #f8f9fa;
            --text-main: #2c3e50;
            --text-muted: #7f8c8d;
            --accent: #3498db;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            line-height: 1.6;
            color: var(--text-main);
            max-width: 1000px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f7f9;
        }
        header {
            text-align: center;
            margin-bottom: 30px;
            padding: 40px 20px;
            background: white;
            border-radius: 16px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.05);
        }
        .summary-nav {
            background: white;
            padding: 20px;
            border-radius: 16px;
            margin-bottom: 30px;
            box-shadow: 0 4px 15px rgba(0,0,0,0.05);
        }
        .summary-nav h3 {
            margin-top: 0;
            font-size: 1.1em;
            color: var(--high-border);
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .summary-list {
            list-style: none;
            padding: 0;
            margin: 0;
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 10px;
        }
        .summary-item {
            font-size: 0.9em;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .item {
            background: var(--interest-bg);
            padding: 25px;
            margin-bottom: 25px;
            border-radius: 16px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.03);
            border-left: 5px solid transparent;
            transition: all 0.2s;
        }
        .item-high {
            background: var(--high-bg);
            border-left-color: var(--high-border);
            box-shadow: 0 6px 18px rgba(243, 156, 18, 0.1);
        }
        .item-uninterested {
            padding: 12px 20px;
            background: #eee;
            opacity: 0.8;
        }
        .item-uninterested .description {
            display: none;
            margin-top: 15px;
            padding-top: 15px;
            border-top: 1px solid #ddd;
        }
        .item-uninterested.expanded .description {
            display: block;
        }
        .item-uninterested h2 {
            font-size: 1.1em;
            margin-bottom: 0;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
        }
        .item-uninterested h2::after {
            content: '展开 +';
            font-size: 0.7em;
            color: var(--accent);
        }
        .item-uninterested.expanded h2::after {
            content: '收起 -';
        }

        .stars {
            color: #f39c12;
            font-size: 1.2em;
            margin-bottom: 5px;
        }
        h2 {
            margin-top: 0;
            margin-bottom: 10px;
            line-height: 1.4;
        }
        h2 a {
            color: var(--text-main);
            text-decoration: none;
        }
        h2 a:hover {
            color: var(--accent);
        }
        .meta {
            font-size: 0.85em;
            color: var(--text-muted);
            margin-bottom: 15px;
        }
        .reason {
            font-size: 0.85em;
            background: rgba(52, 152, 219, 0.08);
            color: #2980b9;
            padding: 6px 12px;
            border-radius: 6px;
            display: inline-block;
            margin-bottom: 15px;
        }
        .description {
            font-size: 0.95em;
            color: #444;
        }
    </style>
</head>
<body>
    <header>
        <h1>{{.SourceTitle}}</h1>
        <div class="report-meta">
            <span>Generated via <strong>{{.SourceName}}</strong></span> | 
            <span>Total Items: <strong>{{.TotalItems}}</strong></span> | 
            <span>{{.GeneratedAt}}</span>
        </div>
    </header>

    <div class="summary-nav">
        <h3>⭐⭐ 深度关注 (Highlights)</h3>
        <ul class="summary-list">
            {{range $index, $item := .Items}}
                {{if eq $item.InterestLevel "high_interest"}}
                <li class="summary-item">
                    <a href="#item-{{$index}}">⭐⭐ {{$item.Title}}</a>
                </li>
                {{end}}
            {{end}}
        </ul>
    </div>

    <main>
        {{range $index, $item := .Items}}
        <div id="item-{{$index}}" class="item 
            {{if eq $item.InterestLevel "high_interest"}}item-high{{end}}
            {{if eq $item.InterestLevel "uninterested"}}item-uninterested{{end}}"
            {{if eq $item.InterestLevel "uninterested"}}onclick="this.classList.toggle('expanded')"{{end}}>
            
            <div class="stars">{{stars $item.InterestLevel}}</div>
            <h2><a href="{{$item.Link}}" target="_blank" onclick="event.stopPropagation()">{{$item.Title}}</a></h2>
            
            <div class="meta">
                <strong>{{$item.Source}}</strong> | {{$item.PubDate}}
            </div>

            {{if $item.Reason}}
            <div class="reason">
                💡 {{$item.Reason}}
            </div>
            {{end}}

            <div class="description">
                {{$item.Description}}
            </div>
        </div>
        {{end}}
    </main>

    <script>
        // Smooth scroll for nav links
        document.querySelectorAll('.summary-item a').forEach(anchor => {
            anchor.addEventListener('click', function (e) {
                e.preventDefault();
                document.querySelector(this.getAttribute('href')).scrollIntoView({
                    behavior: 'smooth'
                });
            });
        });
    </script>
</body>
</html>
`

