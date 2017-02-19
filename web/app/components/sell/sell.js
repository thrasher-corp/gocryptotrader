
angular.module('myApp.sell',[]).component('sell', {
  templateUrl: '/components/sell/sell.html',
  controller:'SellController',
  controller: function ($scope, $http, Notification, $rootScope) {
    $scope.currency = {};
    $scope.exchange = {};

    $rootScope.$on('CurrencyChanged', function (event, args) {
       $scope.currency = args.Currency;
       $scope.exchange = args.Exchange;
       console.log($scope.currency);
       $scope.GetLatestDataFromExchangeCurrency();
        $scope.price = $scope.currency.Bid;
     });

    $scope.GetLatestDataFromExchangeCurrency = function () {
       $http.get('/GetLatestDataFromExchangeCurrency?exhange=' + $scope.exchange.exchangeName + '&currency='+ $scope.currency.CurrencyPair).success(function (data) {
         $scope.currency.Last = data.Last;
         $scope.currency.Volume = data.Volume;
          $scope.currency.Bid = data.Bid;
          $scope.price = $scope.currency.Bid;
       });
     }  

     $scope.placeOrder = function () {
       var obj = {};
       obj.ExchangeName = $scope.exchange.exchangeName;
       obj.Currency = $scope.currency;
       obj.Price = $scope.price;
       obj.Amount = $scope.amount;
       $http.post('/Command/PlaceSellOrder', obj).success(function (response) {
         Notification.success("Successfully placed order");
       });
     };
  }
});



