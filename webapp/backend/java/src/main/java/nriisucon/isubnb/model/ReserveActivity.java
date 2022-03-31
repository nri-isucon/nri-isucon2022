package nriisucon.isubnb.model;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDate;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class ReserveActivity {
    private String id;
    private int userId;
    private int activityId;
    private LocalDate date;
    private int numberOfPeople;
    private boolean isDeleted;
}
