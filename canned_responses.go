package goaspen

var (
	http500Response = []byte(`
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
	http404Response = []byte(`
<!DOCTYPE html>
<html>
  <head>
    <title>404 Not Found</title>
    <style type="text/css">
      body {
        color: #fff;
        background: #555;
      }
      #crying {
        font-size: 48px;
      }
    </style>
  </head>
  <body>
    <h1>404 Not Found</h1>
    <p id="crying">(Ｔ▽Ｔ)</p>
  </body>
</html>
`)
)
