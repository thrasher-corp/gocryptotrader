import { Injectable } from '@angular/core';
import { MatSidenav, MatDrawerToggleResult } from '@angular/material';

@Injectable()
export class SidebarService {
  private sidenav: MatSidenav;

  /**
   * Setter for sidenav.
   *
   * @param {MatSidnav} sidenav
   */
  public setSidenav(sidenav: MatSidenav) {
    this.sidenav = sidenav;
  }

  /**
   * Open this sidenav, and return a Promise that will resolve when it's fully opened (or get rejected if it didn't).
   *
   * @returns Promise<MatSidnavToggleResult>
   */
  public open(): Promise<MatDrawerToggleResult> {
    this.sidenav.open();

    return;
  }

  /**
   * Close this sidenav, and return a Promise that will resolve when it's fully closed (or get rejected if it didn't).
   *
   * @returns Promise<MatSidnavToggleResult>
   */
  public close(): Promise<MatDrawerToggleResult> {
    this.sidenav.close();
    return;
  }

  /**
   * Toggle this sidenav. This is equivalent to calling open() when it's already opened, or close() when it's closed.
   *
   * @param {boolean} isOpen  Whether the sidenav should be open.
   *
   * @returns {Promise<MatSidnavToggleResult>}
   */
  public toggle(isOpen?: boolean): Promise<MatDrawerToggleResult> {
    this.sidenav.toggle(isOpen);
    return;
  }
}
