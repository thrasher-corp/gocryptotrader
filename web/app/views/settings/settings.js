'use strict';

angular.module('myApp.settings', ['ngRoute'])
.config(['$routeProvider', function($routeProvider) {
  $routeProvider.when('/settings', {
    templateUrl: '/views/settings/settings.html',
    controller: 'SettingsController'
  });
}])

.controller('SettingsController', function ($scope, $http, Notification) {
  $scope.getconfigData = function() {
    $http({
      method: 'GET',
      url: '/config/all'
    }).
    success(function (data, status, headers, config) {
      for(var i=0; i<data.Exchanges.length;i++) {
         data.Exchanges[i].AvailablePairsSplit = data.Exchanges[i].AvailablePairs.split(",");
         data.Exchanges[i].EnabledPairsSplit = data.Exchanges[i].EnabledPairs.split(",");
      }
      $scope.config = data;
      Notification.info('Settings loaded');
    }).
    error(function (data, status, headers, config) {
      console.log('error');
    });
  };

  $scope.getconfigData();

$scope.toggleCurrencyToEnabledCurrencies = function(currency, exchange) {
  for(var i=0; i<$scope.config.Exchanges.length;i++) {
    if($scope.config.Exchanges[i].Name == exchange.Name) {
      if(exchange.EnabledPairsSplit.indexOf(currency) > -1) {
        $scope.config.Exchanges[i].EnabledPairsSplit.splice(exchange.EnabledPairsSplit.indexOf(currency),1);
        $scope.config.Exchanges[i].EnabledPairs  = $scope.config.Exchanges[i].EnabledPairs.replace(","+ currency,"");
      } else {
        $scope.config.Exchanges[i].EnabledPairsSplit.push(currency);
        $scope.config.Exchanges[i].EnabledPairs  = $scope.config.Exchanges[i].EnabledPairs + "," + currency;
      }
    }
  }
}

$scope.saveAllSettings = function() {
  $scope.postObject = jQuery.extend(true, {}, $scope.config);
  //Purge any unnecessary post data
  delete $scope.postObject.Webserver;
    for(var i=0; i<$scope.postObject.Exchanges.length;i++) {
      delete $scope.postObject.Exchanges[i].AvailablePairsSplit;
      delete $scope.postObject.Exchanges[i].AvailablePairs;
      delete $scope.postObject.Exchanges[i].BaseCurrencies;
      delete $scope.postObject.Exchanges[i].EnabledPairsSplit;
    }

    //Send to be saved
    $http({
      method: 'POST',
      url: '/config/all/save',
      data: $scope.postObject
    }).
    success(function (data) {
      Notification.success('Saved settings');
    }).
    error(function (data) {
      Notification.error('Save failed');
    });
}
});