
package nriisucon.isubnb.response;

import java.util.List;
import lombok.Data;

@Data
public class ReservationActivityResponse {
    private List<ReserveActivityDetail> reservations;

    public ReservationActivityResponse(List<ReserveActivityDetail> reserveActivities) {
        this.reservations = reserveActivities;
    }
}
