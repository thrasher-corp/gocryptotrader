
angular.module('myApp.buy',[]).component('buy', {
  templateUrl: '/components/buy/buy.html',
  controller:'BuyController',
   bindings: {
      exchange: '=',
      currency:'=' 
  }, controller: function ($scope, $http, Notification) {
     
     $scope.GetLatestDataFromExchangeCurrency = function () {
       $http.get('/GetLatestDataFromExchangeCurrency?exhange=' + $scope.exchange.exchangeName + '&currency='+ $scope.currency.CryptoCurrency).success(function (data) {
         $scope.currency.Last = data.Last;
         $scope.currency.Volume = data.Volume;
       });
     }  

     $scope.placeOrder = function () {
       var obj = {};
       obj.ExchangeName = $scope.exchange.exchangeName;
       obj.Currency = $scope.currency;
       obj.Price = $scope.price;
       obj.Amount = $scope.amount;
       obj.Amount = $scope.amount;
       $http.post('/Command/', obj).success(function (response) {
         Notification.success("Successfully placed order");
       });
     };
  }
});



