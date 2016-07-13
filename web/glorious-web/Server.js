var express = require('express')
  , app = express();  
var request = require('request');
 
 var path = __dirname + '/views/';

app.listen(80, function(){
  console.log('CORS-enabled web server listening on port 80');
});


app.get("/",function(req,res){
  res.sendFile(path + "index.html");
});

app.get("/about",function(req,res){
  res.sendFile(path + "about.html");
});

app.get("/contact",function(req,res){
  res.sendFile(path + "contact.html");
});


app.listen(3000,function(){
  console.log("Live at Port 3000");
});



app.get('/Data/:path', function (req, res) {
  request({
    url :'http://localhost:8080/exchanges/Poloniex/latest/BTC_LTC'
  },function(err, resp, body){
    res.send(body);
  })
  
});