var express = require('express');
var app = express();

// set the view engine to ejs
app.set('view engine', 'ejs');

// use res.render to load up an ejs view file

// index page 
app.get('/', function(req, res) {
    res.render('pages/index', {
    });
});

// setting page 
app.get('/settings', function(req, res) {
    res.render('pages/settings', {
    });
});

// about page 
app.get('/about', function(req, res) {
    res.render('pages/about');
});


app.get('/data/all-enabled-currencies', function (req, res) {
  request({
    url :'http://localhost:9050/exchanges/enabled/latest/all'
  },function(err, resp, body){
    res.send(body);
  })
  
});

app.listen(80, function(){
  console.log('CORS-enabled web server listening on port 80');
});