var express = require('express')
  , app = express();  
var requestify = require('requestify');
var bodyParser = require('body-parser')

var request = require('request'); 
var path = __dirname + '/app/';

app.use("/bower_components", express.static(path + '/bower_components'));
app.use( bodyParser.json() );    


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

app.get('/data/all-enabled-exchange-account-info', function (req, res) {
  request({
    url :'http://localhost:9050/exchanges/enabled/accounts/all'
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



////////////////////////////////////////////////////////
// Posts
///////////////////////////////////////////////////////

app.post('/config/all/save', function(req, res) {
  requestify.post('http://localhost:9050/config/all/save', {
      Data: req.body
  })
  .then(function(response) {
      console.log(response);
      res.send(response.body);
  });
});


var port = process.env.GCT_WEB_PORT || 80;
app.listen(port, function(){
  console.log(`GoCyptoTrader website running! Enter http://localhost:${port}/ into browser`);
});