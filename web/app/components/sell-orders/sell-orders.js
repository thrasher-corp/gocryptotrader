
angular.module('myApp.sellOrders',[]).component('sellorders', {
  templateUrl: '/components/sell-orders/sell-orders.html',
  controller:'SellOrdersController',
  controller: function ($scope, $http, Notification, $rootScope) {
    $scope.currency = {};
    $scope.exchange = {};

    $rootScope.$on('CurrencyChanged', function (event, args) {
       $scope.currency = args.Currency;
       $scope.exchange = args.Exchange;
       $scope.currencyOne = $scope.currency.FirstCurrency;
       $scope.currencyTwo  = $scope.currency.SecondCurrency;
       $scope.getRecentSellOrders();
     });

     $scope.getRecentSellOrders = function() {
       var exchData = {params : {exchangeName: '', currencyPair:''}};
       $http.get('/GetSellOrdersForCurrencyPair' , exchData).success(function(data) {
          $scope.sellOrders = data;
       }).error(function() {
           $scope.sellOrders = [
          {price:456,currencyOneAmount:12,currencyTwoAmount:13,sum:1111},
          {price:234,currencyOneAmount:15,currencyTwoAmount:13,sum:11231},
          {price:12344,currencyOneAmount:232,currencyTwoAmount:13,sum:4511},
          {price:15467,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:6717,currencyOneAmount:2452,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:34522,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
       ];
       });
     }

  }
});



