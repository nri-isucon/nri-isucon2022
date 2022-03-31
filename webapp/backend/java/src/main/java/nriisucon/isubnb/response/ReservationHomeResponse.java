
package nriisucon.isubnb.response;

import java.util.List;
import lombok.Data;

@Data
public class ReservationHomeResponse {

    private List<ReserveHomeDetail> reservations;

    public ReservationHomeResponse(List<ReserveHomeDetail> reserveHomes) {
        this.reservations = reserveHomes;
    }
}
