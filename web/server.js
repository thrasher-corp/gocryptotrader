var express = require('express')
  , app = express();  
var request = require('request'); 
var path = __dirname + '/app/';

app.use("/bower_components", express.static(path + '/bower_components'));

app.get("/",function(req,res){
  res.sendFile(path + "index.html");
});

app.use("/", express.static(path + '/'));

app.get('/data/all-enabled-currencies', function (req, res) {
  request({
    url :'http://localhost:9050/exchanges/enabled/latest/all'
  },function(err, resp, body){
    res.send(body);
  })
});

app.get('/config/all', function (req, res) {
  request({
    url :'http://localhost:9050/config/all'
  },function(err, resp, body){
    res.send(body);
  })
});



app.listen(80, function(){
  console.log('CORS-enabled web server listening on port 80');
});