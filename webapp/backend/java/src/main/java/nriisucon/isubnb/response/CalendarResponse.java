
package nriisucon.isubnb.response;

import java.util.List;
import lombok.Data;

@Data
public class CalendarResponse {

    private List<CalendarDetail> items;

    public CalendarResponse(List<CalendarDetail> calendars) {
        this.items = calendars;
    }
}
