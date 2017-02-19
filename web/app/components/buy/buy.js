
angular.module('myApp.buy',[]).component('buy', {
  templateUrl: '/components/buy/buy.html',
  controller:'BuyController',
  controller: function ($scope, $http, Notification, $rootScope) {
    $scope.currency = {};
    $scope.exchange = {};

    $rootScope.$on('CurrencyChanged', function (event, args) {
       $scope.currency = args.Currency;
       $scope.exchange = args.Exchange;
       console.log($scope.currency);
       $scope.GetLatestDataFromExchangeCurrency();
      $scope.price = $scope.currency.Ask;
     });

    $scope.GetLatestDataFromExchangeCurrency = function () {
       $http.get('/GetLatestDataFromExchangeCurrency?exhange=' + $scope.exchange.exchangeName + '&currency='+ $scope.currency.CurrencyPair).success(function (data) {
         $scope.currency.Last = data.Last;
         $scope.currency.Volume = data.Volume;
          $scope.currency.Ask = data.Ask;
          $scope.price = $scope.currency.Ask;
       });
     }  

     $scope.placeOrder = function () {
       var obj = {};
       obj.ExchangeName = $scope.exchange.exchangeName;
       obj.Currency = $scope.currency;
       obj.Price = $scope.price;
       obj.Amount = $scope.amount;
       $http.post('/Command/PlaceBuyOrder', obj).success(function (response) {
         Notification.success("Successfully placed order");
       });
     };
  }
});



