
angular.module('myApp.buy',[]).component('buy', {
  templateUrl: '/components/buy/buy.html',
  controller:'BuyController',
   bindings: {
    message: '='
  }
}).controller('BuyController', function ($http, Notification) {
  //This contrioller will retrieve all enabled exchanges, 
  //their enabled currencies and the currency's latest ask
  //This call will be used for selling too
  //
  //This will allow a user to make decisions based onthe latest information to them
  //It will auto poll every X seconds (at least until a push method is implemented)
  //When all fields are valid, a purchase order will be sent to and handle by gocryptoServer

  //Could also hard-type the exchange and currency via attributes on the component for quick use
  //Or at lest controlled by passing data from other components/data
});

