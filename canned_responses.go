package goaspen

var (
	HTTP_500_RESPONSE = []byte(`
<!DOCTYPE html>
<html>
  <head>
    <title>500 Internal Server Error</title>
    <style type="text/css">
      body {
        color: #fff;
        background: #f00;
      }
    </style>
  </head>
  <body>
    <h1>500 Internal Server Error</h1>
    <p>Something go boom inside.</p>
  </body>
</html>
`)
)
