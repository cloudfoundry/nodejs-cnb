const fs = require('fs')
var exec = require('child_process').exec;
var logfmt = require("logfmt");
const http = require('http')

const port = process.env.PORT || 8080

const requestHandlerText = (request, response) => {
  const text_1 = fs.readFileSync("text_1.txt")
  response.end(`Text: ${text_1}\n`)
}

const requestHandler = (request, response) => {
    exec('dotnet --version', (error, stdout, stderr) => {
        if (error) {
            response.end('Error: ' + error);
        } else {
            response.end('dotnet: ' + stdout);
        }
    });
}

var middleware = compose([wrapHandler('/', requestHandler),
    wrapHandler('/text', requestHandlerText)]);

const server = http.createServer(middleware).listen(port);

function wrapHandler(path, cb) {
    return function (req, res, next) {
        if (req.url === path) {
            cb(req, res);
        } else {
            next();
        }
    };
}

function notFoundHandler(req, res) {
    res.writeHead(404, { 'Content-Type': 'text/html' });
    res.write('No Path found');
    res.end();
};

// adapted from koa-compose
function compose(middleware) {
    return function (req, res){
        let next = function () {
            notFoundHandler.call(this, req, res);
        };

        let i = middleware.length;
        while (i--) {
            let thisMiddleware = middleware[i];
            let nextMiddleware = next;
            next = function () {
                thisMiddleware.call(this, req, res, nextMiddleware);
            }
        }
        return next();
    }
}

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }

  console.log(`server is listening on ${port}`)
})
