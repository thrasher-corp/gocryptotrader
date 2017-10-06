import { Injectable } from '@angular/core';
import { MdSidenav, MdDrawerToggleResult } from '@angular/material';

@Injectable()
export class SidebarService {
  private sidenav: MdSidenav;

  /**
   * Setter for sidenav.
   *
   * @param {MdSidenav} sidenav
   */
  public setSidenav(sidenav: MdSidenav) {
    this.sidenav = sidenav;
  }

  /**
   * Open this sidenav, and return a Promise that will resolve when it's fully opened (or get rejected if it didn't).
   *
   * @returns Promise<MdSidenavToggleResult>
   */
  public open(): Promise<MdDrawerToggleResult> {
    return this.sidenav.open();
  }

  /**
   * Close this sidenav, and return a Promise that will resolve when it's fully closed (or get rejected if it didn't).
   *
   * @returns Promise<MdSidenavToggleResult>
   */
  public close(): Promise<MdDrawerToggleResult> {
    return this.sidenav.close();
  }

  /**
   * Toggle this sidenav. This is equivalent to calling open() when it's already opened, or close() when it's closed.
   *
   * @param {boolean} isOpen  Whether the sidenav should be open.
   *
   * @returns {Promise<MdSidenavToggleResult>}
   */
  public toggle(isOpen?: boolean): Promise<MdDrawerToggleResult> {
    return this.sidenav.toggle(isOpen);
  }
}