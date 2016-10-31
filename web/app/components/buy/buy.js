
angular.module('myApp.buy',[]).component('buy', {
  templateUrl: '/components/buy/buy.html',
  controller:'BuyController',
   bindings: {
    message: '='
  }
}).controller('BuyController', function ($scope, $http, Notification) {
  alert(this.message);
});

