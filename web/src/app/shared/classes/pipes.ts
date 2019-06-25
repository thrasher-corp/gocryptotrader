import { Component, OnInit, OnDestroy, Pipe, PipeTransform } from '@angular/core';
import { CurrencyPairRedux } from './../../shared/classes/config';


@Pipe({
    name: 'iterateMap'
  })
  export class IterateMapPipe implements PipeTransform {
    transform(iterable: any, args: any[]): any {
      const result = [];

      if (iterable.entries) {
        iterable.forEach((key, value) => {
          result.push({
            key,
            value
          });
        });
      } else {
        for (const key in iterable) {
          if (iterable.hasOwnProperty(key)) {
            result.push({
              key,
              value: iterable[key]
            });
          }
        }
      }

      return result;
    }
  }

  @Pipe({
    name: 'enabledCurrencies'
  })
  export class EnabledCurrenciesPipe implements PipeTransform {
    transform(items: CurrencyPairRedux[], args: any[]): any {
      if (!items) {
        return items;
      }
      return items.filter(item => item.enabled === true);
    }
  }

