
angular.module('myApp.buyOrders',[]).component('buyorders', {
  templateUrl: '/components/buy-orders/buy-orders.html',
  controller:'BuyOrdersController',
  controller: function ($scope, $http, Notification, $rootScope) {
    $scope.currency = {};
    $scope.exchange = {};

    $rootScope.$on('CurrencyChanged', function (event, args) {
       $scope.currency = args.Currency;
       $scope.exchange = args.Exchange;
       $scope.currencyOne = $scope.currency.FirstCurrency;
       $scope.currencyTwo  = $scope.currency.SecondCurrency;
       $scope.doTheThing();
     });

     $scope.doTheThing = function() {
       $scope.buyOrders = [
          {price:12,currencyOneAmount:12,currencyTwoAmount:13,sum:1111},
          {price:13,currencyOneAmount:15,currencyTwoAmount:13,sum:11231},
          {price:14,currencyOneAmount:232,currencyTwoAmount:13,sum:4511},
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
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
          {price:17,currencyOneAmount:22,currencyTwoAmount:13,sum:11212311},
       ];
     }

  }
});



